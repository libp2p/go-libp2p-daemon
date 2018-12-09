#!/bin/sh

tmux new-session -d -s foo 'p2pd'
tmux split-window -v -t 0 'cd ../../ && java p2pd --sock=/tmp/p2pd2.sock'
tmux split-window -h 'sleep 1 && cd ../../ && java p2pc --pathc=/tmp/p2c2.sock --pathd=/tmp/p2pd2.sock --command=ListenForMessage'
tmux split-window -v -t 1 '/bin/bash'
tmux select-layout tile
tmux rename-window 'the dude abides'
tmux attach-session -d