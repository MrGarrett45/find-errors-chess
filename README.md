Pull in your chess games from chess.com (planning to add lichess) and analyze them to find positions where you commonly make mistakes

Under development

ToDo:
1. Add lichess support
2. Add caching so you don't re run stale analysis. Also cache errors
3. Figure out what's going on with discrepancy between local analysis and docker container anaylsis
4. Improve UI
5. Handle scenario for analysis hanging
6. Let users see which openings they struggle against (?)
7. Get google auth out of test mode
8. Let users select to only see innacuracies after x moves. Will allow them to ignore if they purposely gambit or something early.
9. Add in some calculations to give users an idea of how fast analysis will be based on engine settings
10. Jobs table should show which auth0 user kicked off the job
11. Cap number of moves free users can explore (i.e. first 8), let pro users do more
12. Increase DB size for production