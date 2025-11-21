Pull in your chess games from chess.com (planning to add lichess) and analyze them to find positions where you commonly make mistakes

Under development

Done:
1. Pull games from chess.com
2. Save them to DB
3. Run analyses using stockfish

ToDo:
1. Save moves/plys in new table to db
3. Add lichess support
4. Add ability to isolate errors
5. Frontend
6. Add multithreading to backend analysis (further work to do)
7. Make sure closing the db safely 
8. Switch from fmt to log

current centipawn_change looks consistent and correct.

Positive = mover worsened their position (bad move / inaccuracy / blunder depending on threshold).

Negative = mover improved their position (good move).

Use thresholds (e.g. 50, 150, 300) to mark inaccuracy / mistake / blunder