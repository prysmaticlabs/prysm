#!/usr/bin/env bash
: '
# *** Instructions ***

# Build beacon chain for linux
bazel build --platforms=@io_bazel_rules_go//go/toolchain:linux_amd64 //beacon-chain

# Tar the binary
tar czvf /tmp/beacon-chain.tar.gz --directory=bazel-bin/beacon-chain/linux_amd64_stripped beacon-chain

# Copy to cloud storage
gsutil cp /tmp/beacon-chain.tar.gz gs://prysmaticlabs/beacon-chain-deployment.tar.gz

# Create template instance
gcloud compute instance-templates create beacon-chain \
    --project=prysmaticlabs \
    --image-family=debian-9 \
    --image-project=debian-cloud \
    --machine-type=g1-small \
    --preemptible \
    --scopes userinfo-email,cloud-platform \
    --metadata app-location=gs://prysmaticlabs/beacon-chain-deployment.tar.gz \
    --metadata-from-file startup-script=scripts/gcp_startup_script.sh \
    --tags beacon-chain

# Navigate to https://console.cloud.google.com/compute/instanceTemplates/list?project=prysmaticlabs
# and create a new instance group with the template.
'



set -ex

# Talk to the metadata server to get the project id and location of application binary.
PROJECTID=$(curl -s "http://metadata.google.internal/computeMetadata/v1/project/project-id" -H "Metadata-Flavor: Google")
DEPLOY_LOCATION=$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/attributes/app-location" -H "Metadata-Flavor: Google")
EXTERNAL_IP=$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip" -H "Metadata-Flavor: Google")

# Install logging monitor. The monitor will automatically pickup logs send to
# syslog.
curl -s "https://storage.googleapis.com/signals-agents/logging/google-fluentd-install.sh" | bash
service google-fluentd restart &

# Install dependencies from apt
apt-get update
apt-get install -yq ca-certificates supervisor

# Get the application tar from the GCS bucket.
gsutil cp $DEPLOY_LOCATION /app.tar
mkdir -p /app
tar xvzf /app.tar -C /app
chmod +x /app/beacon-chain

# Create a goapp user. The application will run as this user.
getent passwd goapp || useradd -m -d /home/goapp goapp
chown -R goapp:goapp /app

# Configure supervisor to run the Go app.
cat >/etc/supervisor/conf.d/goapp.conf << EOF
[program:goapp]
directory=/app
command=/app/beacon-chain --p2p-host-ip=${EXTERNAL_IP} --clear-db
autostart=true
autorestart=true
user=goapp
environment=HOME="/home/goapp",USER="goapp"
stdout_logfile=syslog
stderr_logfile=syslog
EOF

supervisorctl reread
supervisorctl update

# Application should now be running under supervisor
