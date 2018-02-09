#!/bin/bash

# eventsgateway
# https://github.com/topfreegames/eventsgateway
#
# Licensed under the MIT license:
# http://www.opensource.org/licenses/mit-license
# Copyright © 2018 Top Free Games <backend@tfgco.com>


docker login -u="$DOCKER_USERNAME" -p="$DOCKER_PASSWORD"

echo "sending image to dockerhub"
docker build -t eventsgateway .

# If this is not a pull request, update the branch's docker tag.
if [ $TRAVIS_PULL_REQUEST = 'false' ]; then
  docker tag eventsgateway:latest eventsgateway:${TRAVIS_BRANCH/\//-} \
    && docker push eventsgateway:${TRAVIS_BRANCH/\//-};

  # If this commit has a tag, use on the registry too.
  if ! test -z $TRAVIS_TAG; then
    docker tag eventsgateway:latest $REPO:${TRAVIS_TAG} \
      && docker push $REPO:${TRAVIS_TAG};
  fi
fi
