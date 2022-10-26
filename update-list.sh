#!/bin/bash

set -e

SRC=https://raw.githubusercontent.com/honwen/openwrt-dnsmasq-extra/master/dnsmasq-extra/files/data

rm -rf embed
mkdir -p embed

cd embed
curl -Lo bypassList.gz https://raw.githubusercontent.com/honwen/openwrt-dnsmasq-extra/master/dnsmasq-extra/files/data/direct.gz
curl -Lo tldnList.gz https://raw.githubusercontent.com/honwen/openwrt-dnsmasq-extra/master/dnsmasq-extra/files/data/tldn.gz
curl -Lo specList.gz https://raw.githubusercontent.com/honwen/openwrt-dnsmasq-extra/master/dnsmasq-extra/files/data/gfwlist.lite.gz
cd -

md5sum embed/*.gz
gzip -d embed/*.gz

echo "# Info: delete somesth"
head=$(sed -ne '/router.asus.com/=' embed/bypassList | tail -n 1)
sed "1,${head}d" -i embed/bypassList

sed '/^[0-9\.]*$/d' -i embed/*
md5sum embed/*

echo "# Info: Done"
