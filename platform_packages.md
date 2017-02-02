### Platform Specific Installation Methods


### Arch Linux or Arch based distros.
This includes Arch linux, Antergos, Manjaro, etc. [List](https://wiki.archlinux.org/index.php/Arch_based_distributions_(active))

```sh
yaourt -S drive
```
Since drive is in the aur, you will need an aur helper such as yaourt above. If you are not fimilar with
a helper, you can find a list [here](https://wiki.archlinux.org/index.php/AUR_helpers#AUR_search.2Fbuild_helpers)


### Ubuntu, or Ubuntu based distros. 
This [PPA](https://launchpad.net/~twodopeshaggy/+archive/ubuntu/drive) includes Ubuntu, Mint, Linux Lite, etc. [List](http://distrowatch.com/search.php?basedon=Ubuntu)

```sh
sudo add-apt-repository ppa:twodopeshaggy/drive
sudo apt-get update
sudo apt-get install drive
```

### Debian, or Debian based distros.
You may need to install the package software-properties-common to use apt-add-repository command.

```sh
sudo apt-get install software-properties-common dirmngr
```

After installing software-properties-common, you can run these commands. Updates will be as normal with all debian packages.
Note: The apt-key command is no longer required on apt 1.1 systems. It's safe to ignore any error presented.

```sh
sudo apt-add-repository 'deb http://shaggytwodope.github.io/repo ./'
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv-keys 7086E9CC7EC3233B
sudo apt-key update
sudo apt-get update
sudo apt-get install drive
```

### openSUSE distro. (may also work with CentOS, Red Hat)
```sh
# install needed software tools
sudo yum install go mercurial git hg-git
mkdir $HOME/go
export GOPATH=$HOME/go
# For convenience, add the workspace's bin subdirectory to your PATH:
export PATH=$PATH:$GOPATH/bin

# get and compile the drive program
go get github.com/odeke-em/drive/cmd/drive

# run drive with this command:
$GOPATH/bin/drive
```

### Fedora
Fedora rpms are available from the Fedora Copr project [here](https://copr.fedorainfracloud.org/coprs/vaughan/drive-google/)

Enable the copr repository:

```
dnf copr enable vaughan/drive-google
```

Install the package (drive-google):

```
dnf install drive-google
```

### Packages Provided By

Platform | Author |
---------| -------|
[Arch Linux](https://aur.archlinux.org/packages/drive) | [Jonathan Jenkins](https://github.com/shaggytwodope)
[Ubuntu Linux](https://launchpad.net/~twodopeshaggy/+archive/ubuntu/drive) | [Jonathan Jenkins](https://github.com/shaggytwodope)
[openSUSE Linux]() | [Grant Rostig](https://github.com/grantrostig)
[Fedora](https://copr.fedorainfracloud.org/coprs/vaughan/drive-google/) | [Vaughan Agrez](https://github.com/agrez)

