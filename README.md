# redac

CLI tool for redash

# Installation

```
go install github.com/go-yushi-nakai/redac/cli/redac@v0.1.1
go install github.com/go-yushi-nakai/redac/cli/redac-util@v0.1.1
```


# Setup

```
$ redac-util config add
context name: <Input context name>
redash URL: <Input endpoint of redash>
API Key: <Input your API Key>
list of data sources from ..:
  id=X: source A
  id=Y: source B
select source ID: <Input source ID>

$ redac-util config list
...

$ redac-util config del <context name>
...
```


# Execute query

## Execute query from file

```
$ cat test.sql
select 1;

$ ./redac test.sql <context name>
  1
-----
  1
```


## Execute query from argument

```
$ redac -e 'select "OK" as "This is Test"' <context name>
  THIS IS TEST
----------------
  OK
```
