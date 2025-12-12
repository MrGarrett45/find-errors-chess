import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Chess, type PieceSymbol } from "chess.js";
import type { EngineLine, EngineScore } from "../types";

type WorkerMessage =
  | { type: "info"; data: string }
  | { type: "ready" }
  | { type: "error"; error: string };

type UseStockfishOptions = {
  depth?: number;
  multipv?: number;
  debounceMs?: number;
};

type ParsedInfo = {
  depth: number;
  multipv: number;
  pv: string[];
  score: EngineScore | null;
  nodes?: number;
  nps?: number;
};

const DEFAULT_OPTS: Required<UseStockfishOptions> = {
  depth: 14,
  multipv: 3,
  debounceMs: 300,
};

const parseInfoLine = (line: string): ParsedInfo | null => {
  if (!line.startsWith("info ")) return null;

  const tokens = line.split(/\s+/);
  let depth = 0;
  let multipv = 1;
  let score: EngineScore | null = null;
  let pv: string[] = [];
  let nodes: number | undefined;
  let nps: number | undefined;

  for (let i = 1; i < tokens.length; i += 1) {
    const token = tokens[i];

    if (token === "depth" && tokens[i + 1]) {
      const parsed = Number.parseInt(tokens[i + 1], 10);
      if (!Number.isNaN(parsed)) {
        depth = parsed;
      }
      i += 1;
      continue;
    }

    if (token === "multipv" && tokens[i + 1]) {
      const parsed = Number.parseInt(tokens[i + 1], 10);
      if (!Number.isNaN(parsed)) {
        multipv = parsed;
      }
      i += 1;
      continue;
    }

    if (token === "score" && tokens[i + 1] && tokens[i + 2]) {
      const kind = tokens[i + 1];
      const rawValue = Number.parseInt(tokens[i + 2], 10);
      if (!Number.isNaN(rawValue) && (kind === "cp" || kind === "mate")) {
        score = { type: kind, value: rawValue };
      }
      i += 2;
      continue;
    }

    if (token === "nodes" && tokens[i + 1]) {
      const parsed = Number.parseInt(tokens[i + 1], 10);
      if (!Number.isNaN(parsed)) {
        nodes = parsed;
      }
      i += 1;
      continue;
    }

    if (token === "nps" && tokens[i + 1]) {
      const parsed = Number.parseInt(tokens[i + 1], 10);
      if (!Number.isNaN(parsed)) {
        nps = parsed;
      }
      i += 1;
      continue;
    }

    if (token === "pv") {
      pv = tokens.slice(i + 1);
      break;
    }
  }

  if (pv.length === 0 || depth === 0) {
    return null;
  }

  return { depth, multipv, pv, score, nodes, nps };
};

const toSanLine = (fen: string, pv: string[]): string[] => {
  if (!fen || pv.length === 0) return [];

  try {
    const chess = new Chess(fen);
    const sanMoves: string[] = [];

    for (const move of pv) {
      const parsedMove = move.length >= 4 ? move : null;
      if (!parsedMove) {
        sanMoves.push(move);
        continue;
      }

      const from = parsedMove.slice(0, 2);
      const to = parsedMove.slice(2, 4);
      const promotion =
        parsedMove.length > 4
          ? (parsedMove.slice(4, 5) as PieceSymbol)
          : undefined;
      const result = chess.move({ from, to, promotion });
      sanMoves.push(result?.san ?? move);
    }

    return sanMoves;
  } catch {
    return pv;
  }
};

export function useStockfish(
  fen: string,
  opts: UseStockfishOptions = {}
): {
  lines: EngineLine[];
  isReady: boolean;
  isAnalyzing: boolean;
  error: string | null;
  restart: () => void;
} {
  const [lines, setLines] = useState<EngineLine[]>([]);
  const [isReady, setIsReady] = useState(false);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const workerRef = useRef<Worker | null>(null);
  const fenRef = useRef(fen);

  const { depth, multipv, debounceMs } = useMemo(
    () => ({ ...DEFAULT_OPTS, ...opts }),
    [opts]
  );

  useEffect(() => {
    fenRef.current = fen;
  }, [fen]);

  useEffect(() => {
    const worker = new Worker(
      new URL("../workers/stockfishWorker.ts", import.meta.url)
    );
    workerRef.current = worker;

    const handleMessage = (event: MessageEvent<WorkerMessage>) => {
      const payload = event.data;
      if (!payload) return;

      if (payload.type === "ready") {
        setIsReady(true);
        return;
      }

      if (payload.type === "error") {
        setError(payload.error);
        setIsAnalyzing(false);
        return;
      }

      if (payload.type === "info") {
        const text = payload.data;
        if (text === "uciok") {
          setIsReady(true);
        }

        if (text.startsWith("bestmove")) {
          setIsAnalyzing(false);
        }

        const parsed = parseInfoLine(text);
        if (parsed) {
          const san = toSanLine(fenRef.current, parsed.pv);
          setLines((prev) => {
            const next = [...prev];
            const idx = next.findIndex(
              (line) => line.multipv === parsed.multipv
            );
            const updated: EngineLine = {
              multipv: parsed.multipv,
              depth: parsed.depth,
              pv: parsed.pv,
              san,
              score: parsed.score ?? next[idx]?.score ?? null,
              nodes: parsed.nodes ?? next[idx]?.nodes,
              nps: parsed.nps ?? next[idx]?.nps,
            };

            if (idx >= 0) {
              next[idx] = updated;
            } else {
              next.push(updated);
            }

            return next.sort((a, b) => a.multipv - b.multipv);
          });
        }
      }
    };

    const handleError = (evt: ErrorEvent) => {
      setError(evt.message || "Stockfish worker error");
      setIsAnalyzing(false);
    };

    worker.addEventListener("message", handleMessage);
    worker.addEventListener("error", handleError);

    return () => {
      worker.removeEventListener("message", handleMessage);
      worker.removeEventListener("error", handleError);
      worker.terminate();
      workerRef.current = null;
    };
  }, []);

  const requestAnalysis = useCallback(() => {
    const worker = workerRef.current;
    if (!worker || !fen) return;

    setIsAnalyzing(true);
    setError(null);
    setLines([]);
    worker.postMessage({ type: "analyze", fen, depth, multipv });
  }, [fen, depth, multipv]);

  useEffect(() => {
    if (!fen) {
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setLines([]);
      setIsAnalyzing(false);
      return;
    }

    const timer = window.setTimeout(() => {
      requestAnalysis();
    }, debounceMs);

    return () => {
      window.clearTimeout(timer);
      workerRef.current?.postMessage({ type: "stop" });
    };
  }, [fen, requestAnalysis, debounceMs]);

  return { lines, isReady, isAnalyzing, error, restart: requestAnalysis };
}
