# Automate Quick Start Guide for ComplyTime CLI

This quick start guide aims to set up the minimum environment to explore the ComplyTime CLI.
If you intend to run the ComplyTime CLI on a real target system, the content will need to be updated.
Below are two options to quickly start using the ComplyTime CLI.

## Option 1: Run the quick_start.sh in ubi9 container
Assumed that you have installed [git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git) and [Podman](https://podman.io/docs/installation). If your system is Fedora, CentOS or RHEL you could install them directly:

```bash
dnf install git podman -y
```

1. Pull the ComplyTime Git repository to your local system:

```bash
git clone https://github.com/complytime/complytime.git
cd complytime/scripts/quick_start
```

2. Set the RHEL_APPS_REPO variable to the app DNF repository URL:

```bash
export RHEL_APPS_REPO=${RHEL_APPS_REPO} # Change the ${RHEL_APPS_REPO} to the app dnf repo
```

3. Build the container image:

```bash
podman build --build-arg RHEL_APPS_REPO=${RHEL_APPS_REPO} . -t quick-start:latest
podman run -it quick-start:latest /bin/bash
```

## Option 2: Run the quick_start.sh in a fresh RHEL
Assume that you have already installed a fresh RHEL
1. Download or copy the [quick_start.sh](quick_start.sh) script in your fresh RHEL system.

2. Run the script on your fresh RHEL system:

```bash
chmod +x quick_start.sh
export RHEL_APPS_REPO=$RHEL_APPS_REPO # Change the ${RHEL_APPS_REPO} to the app dnf repo
sh quick_start.sh
```

Now you could explore Complytime CLIs:
```bash
complytime list
complytime plan example
complytime generate
complytime scan
```
