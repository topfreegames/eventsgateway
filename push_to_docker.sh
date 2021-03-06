#!/bin/bash

# eventsgateway
# https://github.com/topfreegames/eventsgateway
#
# Licensed under the MIT license:
# http://www.opensource.org/licenses/mit-license
# Copyright © 2018 Top Free Games <backend@tfgco.com>

REPO=tfgco/eventsgateway

docker login -u="$DOCKER_USERNAME" -p="$DOCKER_PASSWORD"

echo "sending image to dockerhub"
docker build -t ${REPO} .

# If this is not a pull request, update the branch's docker tag.
if [ $TRAVIS_PULL_REQUEST = 'false' ]; then
  docker tag ${REPO}:latest ${REPO}:${TRAVIS_BRANCH/\//-} \
    && docker push ${REPO}:${TRAVIS_BRANCH/\//-};

  # If this commit has a tag, use on the registry too.
  if ! test -z $TRAVIS_TAG; then
    docker tag ${REPO}:latest $REPO:${TRAVIS_TAG} \
      && docker push $REPO:${TRAVIS_TAG};
  fi
fi
