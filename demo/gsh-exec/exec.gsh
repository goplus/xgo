xgo "run", "./foo"
exec "xgo run ./foo"
exec "FOO=100 xgo run ./foo"
exec {"FOO": "101"}, "xgo", "run", "./foo"
exec "xgo", "run", "./foo"
exec "ls $HOME"
ls ${HOME}
