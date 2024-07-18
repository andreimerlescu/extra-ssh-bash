# Extra SSH Bash

This project is primarily a bash utility, however it includes a Go and Terraform component to compliment its intended functionality. Currently, the bash utility is not yet implemented, however the concurrent executor offered by the [exec-multi-remote-ssh-bash-cmd](/cmd/main.go) package gives you easy ability to manage a cluster of servers that were created with Terraform.

## Tutorial: Create a Cluster of Docker hosts on EC2 using Terraform using the GitLab Terraform State Management

To use this application with a Docker Cluster, you'll need to get a few things done first: 

0. Clone the repository:

```bash
mkdir -p ~/work/github
cd ~/work/github
git clone git@github.com:andreimerlescu/extra-ssh-bash.git
```

1. Create your Terraform project:

```bash
mkdir -p ~/work/terraform/docker-cluster
cd ~/work/terraform/docker-cluster
```

2. Create the `docker-cluster.tf` file:

```bash
cp ~/work/github/extra-ssh-bash/terraform/docker-cluster/docker-cluster.tf ~/work/terraform/docker-cluster/docker-cluster.tf
vi ~/work/terraform/docker-cluster/docker-cluster.tf
```

Within this file, you'll need to find the following and replace `__REPLACE__` with the digits of the project: 

```bash
PROJECT_ID=2
sed -i "s/PROJECT_ID/${PROJECT_ID}/gI" ~/work/terraform/docker-cluster/docker-cluster.tf
```

To change the URL of the GitLab API:

```bash
API_URL="gitlab.mydomain.com"
sed -i "s/gitlab\.com/$API_URL/gI" ~/work/terraform/docker-cluster/docker-cluster.tf
```

To change the default name of the Terraform state, run (keep -tfstate suffix):

```bash
TF_STATE_NAME=docker-cluster-tfstate
sed -i "s/docker-cluster-tfstate/$TF_STATE_NAME/gI" ~/work/terraform/docker-cluster/docker-cluster.tf
```

Now in your Terminal, lets create an SSH key to use: 

```bash
! [[ -f "${HOME}/.ssh/docker-cluster-master.pem" ]] && ssh-keygen -t ed25519 -N '' -f "${HOME}/.ssh/docker-cluster-master" && mv ~/.ssh/docker-cluster-master ~/.ssh/docker-cluster-master.pem
```

Then, you'll need to replace the public key with something real...

```bash
SSH_PUBKEY="$(cat ~/.ssh/docker-cluster-master.pub | sed 's/[&/\]/\\&/g')"
sed -i "s/REPLACE_WITH_SSH_PUBLIC_KEY/$SSH_KEY/gI" ~/work/terraform/docker-cluster/docker-cluster.tf
```

To change the AWS Region from `us-east-1d` to something like `us-west-2c`: 

```bash
TERRAFORM_REGION="${AWS_REGION:-"${AWS_DEFAULT_REGION:-"us-east-1d"}"}"
sed -i "s/us-east-1d/$TERRAFORM_REGION/gI" ~/work/terraform/docker-cluster/docker-cluster.tf
```

To change the EC2 instance size from `t3.nano` to `c6.xlarge`: 

```bash
EC2_SIZE="c6.large"
sed -i "s/t3\.nano/${EC2_SIZE//\./\\.}/gI" docker-cluster.tf
```

To change the Operating System from `Ubuntu 24.04 LTS`, replace the AMI value: 

```bash
EC2_AMI="ami-01ed6e3767aa5ab34" # Rocky-9-EC2-LVM-9.4-20240523.0.x86_64
sed -i "s/ami-04a81a99f5ec58529/${EC2_AMI}/gI" docker-cluster.tf
```

> **NOTE**: When changing the underlying operating system from something that uses `ufw` and `apt-get` to manage the system to `firewall-d` and `yum`, you may end up breaking downstream compatibility with the originally provided `user_data` of the [docker-cluster.tf](/terraform/docker-cluster/docker-cluster.tf) template. You were warned!

To change the subnet that you want AWS EC2 to put your instance inside, run:

```bash
EC2_SUBNET="subnet-" # REPLACE WITH YOUR AWS SUBNET
sed -i "s/subnet-REPLACEME/${EC2_SUBNET}/gI" ~/work/terraform/docker-cluster/docker-cluster.tf
```

To change the AWS Security Groups: 

```bash
GROUPS=(sg-1 sg-2 sg-3 sg-4 sg-5) # MAX 5 GROUPS PER EC2 INSTANCE PER AWS POLICY
IDX=-1 && for group in "${GROUPS[@]}"; do sed -i "s/SG$((IDX++))-REPLACEME/${group}/gI" ~/work/terraform/docker-cluster/docker-cluster.tf; done
```

Once that's done, you'll need to create your infrastructure: 

If you don't have the AWS CLI installed, you can do so easily: 

```bash
# Installing AWS CLI
cd tmp; rm -rf awscli/*; rmdir awscli; mkdir -p awscli; cd awscli
# Downloading Linux x86_64 Installer
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
# Extract Downloaded File
unzip awscliv2.zip
# Install AWS CLI
sudo ./aws/install
echo "Installed AWS CLI at $(which aws): $(aws version)"
```

Once you have that configured, you'll need to get access to AWS: 

```bash
aws configure
```

Once configured, test your access:

```bash
aws sts get-caller-identity
```

If you're good to go, you should see your `arn` in the output. Press `q` to exit the output of the command.

Next you'll need to initialize the Terraform project `docker-cluster` to create the `docker-cluster-tfstate` in GitLab on your account via the `--token <GitLab Access Token>`. Since the `docker-cluster.tf` script 

```bash
terraform -chdir="~/work/terraform/docker-cluster" init
terraform -chdir="~/work/terraform/docker-cluster" plan
terraform -chdir="~/work/terraform/docker-cluster" apply
terraform -chdir="~/work/terraform/docker-cluster" output -json public_ips
```

Once your infrastructure is running (it'll take a few minutes for the user_data to initialize), you can begin using this project to interact with that cluster and concurrently execute bash commands on all of your terraform members.

```bash
go run . \
    --api "https://gitlab.com/api/v4" \
    --id 3 \
    --token "$(cat "${HOME}/.secrets/docker-cluster-access-token")" \
    --tfdir ~/work/terraform/docker-cluster \
    --user ubuntu \
    --key ~/.ssh/docker-cluster-master.pem \
    --bash "docker ps" \
    --json
```

Returns: 

```json
{
  "66.77.88.99": {
    "cmd": "ssh -i /home/andrei/.ssh/docker-cluster-master.pem -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o CheckHostIP=no ubuntu@66.77.88.99 docker ps",
    "stdout": "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES\n",
    "stderr": ""
  },
  "55.66.77.88": {
    "cmd": "ssh -i /home/andrei/.ssh/docker-cluster-master.pem -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o CheckHostIP=no ubuntu@55.66.77.88 docker ps",
    "stdout": "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES\n",
    "stderr": ""
  },
  "44.55.66.77": {
    "cmd": "ssh -i /home/andrei/.ssh/docker-cluster-master.pem -o IdentitiesOnly=yes -o StrictHostKeyChecking=no -o CheckHostIP=no ubuntu@44.55.66.77 docker ps",
    "stdout": "CONTAINER ID   IMAGE     COMMAND   CREATED   STATUS    PORTS     NAMES\n",
    "stderr": ""
  }
}
```

> **NOTE**: It is **NOT RECOMMENDED** to execute `--bash ""` commands that include functions like `yes` or `tail` or `watch` or any other blocking until-stopped running processes as this will result in unexpected behavior due to the concurrency nature of the runtime. 

> **NOTE**: YOU **CANNOT USE `|` INSIDE `--bash ""`!
