# game-backend

This repository consists of the services needed to matchmake and process game events for a word battle game.

To run services locally, use `./dev/run.sh` to run services locally on your machine. Requires Go >=1.12 and Docker to be installed.

To test services, use `./dev/test.sh`. Requires Go >=1.24 (due to use of testing.T's Context addition from 1.24) and Docker to be installed.