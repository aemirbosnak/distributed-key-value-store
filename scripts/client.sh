#!/bin/bash

case $1 in
    get)
        KEY=${2:?key not given}
        curl "localhost:3000/get?key=$KEY"
        ;;
    put)
        KEY=${2:?key not given}
        VALUE=${3:?value not given}
        curl "localhost:3000/put?key=$KEY&val=$VALUE"
        ;;
    delete)
        KEY=${2:?key not given}
        curl "localhost:3000/delete?key=$KEY"
        ;;
    status)
        curl "localhost:3000/status"
        ;;
    *)
        echo "Unknown usage: $1"
        echo "Valid commands:"
        echo "  get [key]"
        echo "  put [key] [value]"
        echo "  delete [key]"
        echo "  status"
        exit 1
        ;;
esac

