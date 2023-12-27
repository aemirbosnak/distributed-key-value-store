#!/bin/bash

PORT=${1:?no port given}

sudo ss -lptn "sport = :$PORT"

