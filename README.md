# Grafter
`Grafter` is a program designed to stash away all the directories and files you consider archived.

Meant as an archival tool.

`grafter [file1] [file2] ... [fileN]` will add files to the graft.

```
Usage of grafter:
  -copy
        Copy the files of the listed IDs to destdir.
  -destdir string
        Directory to copy the selected files to.
  -exist
        If -exist=1 the files will be grafted to the list. (default true)
  -fetch
        Validate and fetch missing files in the user graft.
  -graftdir string
        Directory to graft the listed files to
  -list
        List everything in the user graft.
  -remove
        Remove the listed IDs from the user graft.
  -version
        Show the current version of grafter.
```
