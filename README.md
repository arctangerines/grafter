# Grafter
`Grafter` is a program designed to stash away all the directories and files you consider archived.

Meant as an archival tool.

`grafter [file1] [file2] ... [fileN]` will add files to the graft.

You can build with by running `make` and/or `make install`
```
Usage of grafter:
  Usage of grafter:
  -c    [Copy] Copy the files of the listed IDs to destdir.
  -d string
        [CopyDir] Directory to copy the selected files to.
  -e    [Exist] If -exist=0 the files will only be grafted to the list. (default true)
  -f    [Fetch] Validate and fetch missing files in the user graft.
  -g string
        [GraftDir] Directory to graft the listed files to
  -l    [List] List everything in the user graft.
  -m    [Move] If the file should be moved to the graft directory. (default true)
  -r    [Remove] Remove the listed IDs from the user graft.
  -v    [verbose] Show the current version of grafter.
```
