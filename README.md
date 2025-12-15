# Grafter
`Grafter` is a program designed to stash away all the directories and files you consider archived.

Meant as an archival tool.

`grafter [file1] [file2] ... [fileN]` will add files to the graft.
`grafter -l` lists files.
`grafter -r [ID1] [ID2] ... [IDN]` will remove the files from the graft.

A `grafts.json` file is created will all the info relevant to the graphs.
The default folder for `Grafter` is `~/Grafts`
