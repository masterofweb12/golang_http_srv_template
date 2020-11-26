#!/bin/sh


CDIR=$PWD
cd $CDIR
echo $CDIR


if [ -f ./http_srv.bin  ]; then
rm ./http_srv.bin
fi

if [ -d $CDIR/logs ]; then

  echo " DIR $CDIR/logs finded"

else

    if [ -f $CDIR/logs  ]; then

          rm $CDIR/logs
    fi

    echo " creating DIR $CDIR/logs"
    mkdir $CDIR/logs
fi


go build -o http_srv.bin .





