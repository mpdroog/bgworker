BGWorker
==================
Run cronjobs/scripts from the webbrowser by calling them through
this small daemon.

How is security enforced?
==================
- The daemon runs as a limited user, example systemd-config sets to user `script:script`
- Because files are directly called thus the execution-flag and shebang are necessary

How does it work?
===================
> DevNote: Don't forget to properly urlescape the arguments for file and arg ;)

- HTTP call to `/queue/add` with `?file=coolscript.sh&args=-write` -> returns JSON `{id:"uniqueidforscriptbeingrun"}`e;
- If you want to keep track, keep polling `/queue/status?id=<ID>` -> return JSON `{status:"PENDING|DONE|ERROR", stdout:"output in stdout and stderr"}` 

TODO
==================
* Add support for N-workers
* Add support for multiple directories (different rules?)
* Add support for streaming stdout/stderr from script to API?
