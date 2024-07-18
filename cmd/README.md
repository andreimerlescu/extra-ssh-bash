# Execute Multi-Remote SSH Bash Commands

As the name suggests, this package is designed to work with Terraform that is using an `http` backend of **GitLab API**. You can either use GitLab Community Edition or GitLab Enterprise or GitLab.com with this application. 

When connecting to the **GitLab API**, you'll need to pass in the `--api` a value such as `https://gitlab.com/api/v4` or `https://gitlab.localdomain:4000/api/v4` along with the `--id` of the Project. 

The terraform project, in the example `~/work/terraform/docker-cluster` directory, is required to either have an `output "public_ips"` that returns a tuple of the ***n***-instance public IP address or have `output "public_ip"` that returns a string of the public IP address of the instance. The output of the `terraform -chdir="<--tfdir>" output -json public_ips` or `terraform -chdir="<--tfdir>" output public_ip` is then used to execute `--bash ""` concurrently against each IP address found. The `-json` flag added at the end renders the output in JSON instead of a text table.

## Usage

```bash
./exec-multi-remote-ssh-bash-cmd --help
```

```log
Usage of ./exec-multi-remote-ssh-bash-cmd:
  -api string
        GitLab API URL (default "https://gitlab.com/api/v4")
  -bash string
        Bash command to execute remotely
  -id int
        GitLab Project ID (default 1)
  -ipcsv string
        CSV string of IP addresses
  -json
        Use JSON formatted output
  -key string
        Path to SSH key for remote access (default ".ssh/id_ed25519")
  -stderr string
        Path to STDERR to write to (default "logs/go.ebs.stderr")
  -stdout string
        Path to STDOUT to write to (default "logs/go.ebs.stdout")
  -tfdir string
        Path to terraform directory (default "terraform")
  -tfoutputvar string
        Output variable name from Terraform to get IP addresses of target hosts (default "public_ips")
  -token string
        GitLab API Access Token
  -user string
        Username of remote host (default "ubuntu")
```
