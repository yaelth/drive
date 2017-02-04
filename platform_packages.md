### Platform Specific Installation Methods

---
### Arch Linux or Arch based distros.
This includes Arch linux, Antergos, Manjaro, etc. [List](https://wiki.archlinux.org/index.php/Arch_based_distributions_(active))

```sh
yaourt -S drive
```
Since drive is in the aur, you will need an aur helper such as yaourt above. If you are not fimilar with
a helper, you can find a list [here](https://wiki.archlinux.org/index.php/AUR_helpers#AUR_search.2Fbuild_helpers)

---
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

---
### Fedora distro.
Fedora rpms are available from the Fedora Copr project [here](https://copr.fedorainfracloud.org/coprs/vaughan/drive-google/)

Enable the copr repository:

```
dnf copr enable vaughan/drive-google
```

Install the package (drive-google):

```
dnf install drive-google
```

---
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

---
### Ubuntu, or Ubuntu based distros. 
This [PPA](https://launchpad.net/~twodopeshaggy/+archive/ubuntu/drive) includes Ubuntu, Mint, Linux Lite, etc. [List](http://distrowatch.com/search.php?basedon=Ubuntu)

```sh
sudo add-apt-repository ppa:twodopeshaggy/drive
sudo apt-get update
sudo apt-get install drive
```

### Automation on the latest stable Ubuntu distribution. 
This GitLab-hosted [PPA](https://gitlab.com/jean-christophe-manciot/ppa) offers:
- the latest compiled drive client
- scripts for automated drive commands including syncing
- many other goodies.

```sh
sudo apt-key adv --keyserver keyserver.ubuntu.com --recv 5F0C7CD8
sudo sh -c $'echo "deb https://gitlab.com/jean-christophe-manciot/ppa/raw/master/Ubuntu yakkety stable #JC Manciot\'s Stable PPA" >> /etc/apt/sources.list.d/jean-christophe-manciot.list'
sudo apt-get update
sudo apt-get install drive-google
```

You may need to install Comodo RSA Domain Validation Secure Server Certificate Authority used by Gitlab. Cf. the [PPA](https://gitlab.com/jean-christophe-manciot/ppa) for details.

An unstable channel also exists:
```sh
sudo sh -c $'echo "deb https://gitlab.com/jean-christophe-manciot/ppa/raw/master/Ubuntu yakkety unstable #JC Manciot\'s Unstable PPA" >> /etc/apt/sources.list.d/jean-christophe-manciot.list'
```

---
### Packages Provided By

Platform | Author |
---------| -------|
[Arch Linux](https://aur.archlinux.org/packages/drive) | [Jonathan Jenkins](https://github.com/shaggytwodope)
[Debian Linux](http://shaggytwodope.github.io/repo) | [Jonathan Jenkins](https://github.com/shaggytwodope)
[Fedora Linux](https://copr.fedorainfracloud.org/coprs/vaughan/drive-google/) | [Vaughan Agrez](https://github.com/agrez)
[openSUSE Linux]() | [Grant Rostig](https://github.com/grantrostig)
[Ubuntu Linux](https://launchpad.net/~twodopeshaggy/+archive/ubuntu/drive) | [Jonathan Jenkins](https://github.com/shaggytwodope)
[Automation on Ubuntu Linux](https://gitlab.com/jean-christophe-manciot/ppa) | [Jean-Christophe Manciot](https://gitlab.com/jean-christophe-manciot)

