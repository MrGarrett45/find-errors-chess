Pull in your chess games from chess.com (planning to add lichess) and analyze them to find positions where you commonly make mistakes

Under development

ToDo:
1. Add lichess support
2. Figure out what's going on with discrepancy between local analysis and docker container anaylsis
3. Improve UI
4. Handle scenario for analysis hanging
5. Let users see which openings they struggle against (?)
6. Get google auth out of test mode
7. Let users select to only see innacuracies after x moves. Will allow them to ignore if they purposely gambit or something early.
8. Add in some calculations to give users an idea of how fast analysis will be based on engine settings
9. Jobs table should show which auth0 user kicked off the job
10. Cap number of moves free users can explore (i.e. first 8), let pro users do more
11. Increase DB size for production
12. Add rate limiting for running analysis
13. Let users choose which kinds of games they want to ingest (blitz, rapid, etc)