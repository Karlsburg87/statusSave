#!/usr/bin/env bash
#Process mangement: https://www.digitalocean.com/community/tutorials/how-to-use-bash-s-job-control-to-manage-foreground-and-background-processes

#Find if programs are installed using command built-in function
#See: http://manpages.ubuntu.com/manpages/trusty/man1/bash.1.html (search: 'Run  command  with  args')
if [[ -x "$(command -v podman)" ]]; then

    podman build -f Dockerfile -t savestatus \
    --build-arg PROJECT_ID="${PROJECT_ID}" \
    --build-arg HOST="${HOST}" \
    --build-arg USERNAME="${USERNAME}" \
    --build-arg PASSWORD="${PASSWORD}" \
    --build-arg DATABASE_NAME="${DATABASE_NAME}" \
    --build-arg ROUTING_ID="${ROUTING_ID}" \
    --build-arg DB_PORT="${DB_PORT}" \
    --build-arg DATABASE_URL="${DATABASE_URL}" \
    --build-arg PORT="${PORT}" \
    .
    
    podman run --rm --name saveStatus -t -p 8080:"${PORT}" savestatus

    #cleanup
    #podman image prune

elif [[ -x "$(command -v docker)" ]]; then
  
    docker build -f Dockerfile -t savestatus \
    --build-arg PROJECT_ID="${PROJECT_ID}" \
    --build-arg HOST="${HOST}" \
    --build-arg USERNAME="${USERNAME}" \
    --build-arg PASSWORD="${PASSWORD}" \
    --build-arg DATABASE_NAME="${DATABASE_NAME}" \
    --build-arg ROUTING_ID="${ROUTING_ID}" \
    --build-arg DB_PORT="${DB_PORT}" \
    --build-arg DATABASE_URL="${DATABASE_URL}" \
    --build-arg PORT="${PORT}" \
    .
    
    docker run --rm --name saveStatus -t -p 8080:"${PORT}" savestatus

    #cleanup
    docker image prune
else
  echo "You need to have either Docker Desktop or Podman to run"
fi