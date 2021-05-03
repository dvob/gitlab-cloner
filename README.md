# gitlab-cloner
Clones a whole group structure into the current directory.

For example you have the following strucutre on Gitlab:
```
root_group/project1
root_group/sub_group1/project2
root_group/sub_group2/sub_group3/project3
```
Then `gitlab-cloner` can be invoked as follows:
```
gitlab-cloner clone root_group
```

Which produces the follwoing local directory structure:
```
$ find -name ".git"
./project1/.git
./sub_group1/project2/.git
./sub_group2/sub_group3/project3/.git
```

## Installation
```
go install github.com/dvob/gitlab-cloner@latest
```
