#!/usr/bin/env bash

BASEURL="https://weather.psg.nexstardigital.net"
ENDPT="/service/api/v3/alerts/getLiveAlertsByCounties"
QP="?counties=25013,25015,25011,25003"
#QP=""

curl "${BASEURL}${ENDPT}${QP}" \
   -H 'accept: */*' \
   -H 'accept-language: en-US,en;q=0.9' \
   -H 'origin: https://www.wwlp.com' \
   -H 'priority: u=1, i' \
   -H 'referer: https://www.wwlp.com/' \
   -H 'sec-ch-ua: "Brave";v="143", "Chromium";v="143", "Not A(Brand";v="24"' \
   -H 'sec-ch-ua-mobile: ?0' \
   -H 'sec-ch-ua-platform: "macOS"' \
   -H 'sec-fetch-dest: empty' \
   -H 'sec-fetch-mode: cors' \
   -H 'sec-fetch-site: cross-site' \
   -H 'sec-gpc: 1' \
   -H 'user-agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36'


