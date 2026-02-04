Pull in your chess games from chess.com/lichess and analyze them to find positions where you commonly make mistakes

Under development

ToDo:
1. Figure out what's going on with discrepancy between local analysis and docker container anaylsis
2. Improve UI
3. Let users see which openings they struggle against (?)
4. Get google auth out of test mode
5. Let users select to only see innacuracies after x moves. Will allow them to ignore if they purposely gambit or something early.
6. Add in some calculations to give users an idea of how fast analysis will be based on engine settings
8. Cap number of moves free users can explore (i.e. first 8), let pro users do more
9. Increase DB size for production
10. Add rate limiting for running analysis
11. Let users choose which kinds of games they want to ingest (blitz, rapid, etc)
12. Figure out why depth analysis is so slow
13. Figure out fail isn't recorded as happening on frontend when engine times out