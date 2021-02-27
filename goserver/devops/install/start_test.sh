#! /bin/sh

ROOTPATH="/data/go_micro"

MODE="test"
if [[ $1 = "dev" ]]; then
    MODE="dev"
fi

mkdir -p ${ROOTPATH}_run/log

function start_program()
{
    local prog=$1
    local prog_name=${prog}_server
    local prog_path=$ROOTPATH/src/$prog_name

    killall -9 ${ROOTPATH}_run/$prog_name
    cd $prog_path
    go build
    \cp -f $prog_path/$prog_name ${ROOTPATH}_run/$prog_name
    nohup ${ROOTPATH}_run/$prog_name  -config=$prog_path/${prog}-${MODE}.toml >> ${ROOTPATH}_run/log/${prog_name}.log 2>&1 &
}
start_program config
sleep 4
start_program api
sleep 1
