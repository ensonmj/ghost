#!/bin/sh

wget --no-check-certificate -qO - https://easylist-downloads.adblockplus.org/easylistchina+easylist.txt | grep -P '^\|\|[^\*]+\^$' | sed -e 's:||:address\=\/:' -e 's:\^:/127\.0\.0\.1:' > /tmp/easylist.conf

wget --no-check-certificate -qO - https://easylist-downloads.adblockplus.org/easyprivacy.txt | grep -P '^\|\|[^\*]+\^$' | sed -e 's:||:address\=\/:' -e 's:\^:/127\.0\.0\.1:' > /tmp/easyprivacy.conf

# line end with \r\n
wget --no-check-certificate -qO - https://raw.githubusercontent.com/cjx82630/cjxlist/master/cjx-annoyance.txt | grep -P '^\|\|[^\*]+\^$' | sed -e 's:||:address\=\/:' -e 's:\^:/127\.0\.0\.1:' > /tmp/cjxlist.conf
