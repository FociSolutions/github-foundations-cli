# Github Foundations CLI
A command-line tool for the Github Foundations framework.

## Table of Contents

- [Usage](#usage)
    - [Generate](#generate)
    - [Import](#import)
    - [Check](#check)
    - [List](#list)
    - [Help](#help)
- [Installation](#installation)
    - [From releases](#from-releases)
        - [Linux](#linux)
        - [MacOS](#macos)
        - [Windows](#windows)
    - [From source](#from-source)







## Usage

There are a few main tools provided by the Github Foundations CLI:

```
Usage:
    gh_foundations [command]

Available Commands:
    gen         Generate HCL input for GitHub Foundations.
    import      Starts an interactive import process for resources in a Terraform plan.
    check       Perform checks against a Github configuration.
    list        List various resources managed by the tool.
    help        Help about any command.

Flags:
    -h, -- help     help for gh_foundations
```

### Generate

This command is used to generate HCL input for GitHub Foundations. This tool is used to generate HCL input for GitHub Foundations from state files output by terraformer.

```
Usage:
    gh_foundations gen <resource>
```

Where `<resource>` is one of the following:
- `repository`

### Import

This command will start an interactive process to import resources into Terraform state. It uses the results of a terraform plan to determine which resources are available for import.
    
```
Usage:
    gh_foundations import [module_path]

```

Where `<module_path>` is the path to the Terragrunt module to import.

### Check

Perform checks against a Github configuration and generate reports. This is used to validate the compliance stance of your GitHub configuration.

```
    Usage:
    gh_foundations check <org-slug>

```

Where `<org-slug>` is the organization slug to check.

### List

list various resources managed by the tool.


```
    Usage:
    gh_foundations list <resource> [ProjectDirectory] [options]

```

Where `<resource>` is one of the following:
- repos
- orgs

`[ProjectDirectory]` is the path to the Terragrunt `Project` directory.

`[options]` is a list of options to filter the list of resources. The options are:
- repos:
    - `--ghas`, `-g`    List repositories with GHAS enabled.

### Help

Display help for the tool.

## Installation

### From releases
Download the latest release from the [releases page](http:github.com/FociSolutions/github-foundations-cli/releases) and run the following commands:


#### Linux

**ADM64**
```
curl -LO "https://github.com/FociSolutions/github-foundations-cli/releases/download/$(curl -s https://api.github.com/repos/FociSolutions/github-foundations-cli/releases/latest | grep tag_name | cut -d '"' -f 4)/UPDATE_ME_github_foundations_linux_amd64"
chmod +x UPDATE_ME_github_foundations_linux_amd64
sudo mv UPDATE_ME_github_foundations_linux_amd64 /usr/local/bin/gh_foundations
```

**ARM64**
```
curl -LO "https://github.com/FociSolutions/github-foundations-cli/releases/download/$(curl -s https://api.github.com/repos/FociSolutions/github-foundations-cli/releases/latest | grep tag_name | cut -d '"' -f 4)/UPDATE_ME_github_foundations_linux_arm64"
chmod +x UPDATE_ME_github_foundations_linux_arm64
sudo mv UPDATE_ME_github_foundations_linux_arm64 /usr/local/bin/gh_foundations
```

#### MacOS

**ADM64**
```
curl -LO "https://github.com/FociSolutions/github-foundations-cli/releases/download/$(curl -s https://api.github.com/repos/FociSolutions/github-foundations-cli/releases/latest | grep tag_name | cut -d '"' -f 4)/UPDATE_ME_github_foundations_darwin_amd64"
chmod +x UPDATE_ME_github_foundations_darwin_amd64
sudo mv UPDATE_ME_github_foundations_darwin_amd64 /usr/local/bin/gh_foundations
```

**ARM64**
```
curl -LO "https://github.com/FociSolutions/github-foundations-cli/releases/download/$(curl -s https://api.github.com/repos/FociSolutions/github-foundations-cli/releases/latest | grep tag_name | cut -d '"' -f 4)/UPDATE_ME_github_foundations_darwin_arm64"
chmod +x UPDATE_ME_github_foundations_darwin_arm64
sudo mv UPDATE_ME_github_foundations_darwin_arm64 /usr/local/bin/gh_foundations
```

#### Windows

**i386**
```
curl -LO "https://github.com/FociSolutions/github-foundations-cli/releases/download/$(curl -s https://api.github.com/repos/FociSolutions/github-foundations-cli/releases/latest | grep tag_name | cut -d '"' -f 4)/UPDATE_ME_github_foundations_windows_386.exe"
...
```

**ADM64**
```
curl -LO "https://github.com/FociSolutions/github-foundations-cli/releases/download/$(curl -s https://api.github.com/repos/FociSolutions/github-foundations-cli/releases/latest | grep tag_name | cut -d '"' -f 4)/UPDATE_ME_github_foundations_windows_amd64.exe"
...
```

**ARM64**
```
curl -LO "https://github.com/FociSolutions/github-foundations-cli/releases/download/$(curl -s https://api.github.com/repos/FociSolutions/github-foundations-cli/releases/latest | grep tag_name | cut -d '"' -f 4)/UPDATE_ME_github_foundations_windows_arm64.exe"
... 
```

### From source
1.  Run `git clone <gh_foundations_cli repo> && cd gh-foundations-cli/`
2.  Run `go mod download`
3.  Run `go build -v` for all providers OR build with one provider

