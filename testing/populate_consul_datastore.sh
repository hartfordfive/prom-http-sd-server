#!/bin/bash

TARGET_GROUP1=nginx_webserver
TARGET_GROUP1_HOSTS="10.0.4.101:9555 10.0.4.102:9555 10.0.4.103:9555 10.0.4.104:9555 10.0.4.105:9555"
TARGET_GROUP1_LABELS="labels=__meta_type=infrastructure&labels=__meta_virtualization=vmware&labels=__meta_server_type=virtual"

TARGET_GROUP2=haproxy
TARGET_GROUP2_HOSTS="10.0.3.101:9500 10.0.3.102:9500 10.0.3.103:9500"
TARGET_GROUP2_LABELS="labels=__meta_type=infrastructure&labels=__meta_server_type=physical"

TARGET_GROUP3=node_exporter
TARGET_GROUP3_HOSTS="10.0.2.100:9600 10.0.2.101:9600 10.0.2.102:9600 10.0.2.103:9600 10.0.2.104:9600 10.0.2.105:9600 10.0.2.106:9600"
TARGET_GROUP3_LABELS="labels=__meta_type=infrastructure&labels=__meta_server_os=slinux"

TARGET_GROUP4=network_switches_amer # SNMP via snmp_exporter
TARGET_GROUP4_HOSTS="10.10.1.101:60 10.10.1.102:60 10.10.1.103:60 10.10.1.104:60 10.10.1.105:60 10.10.1.106:60"
TARGET_GROUP4_LABELS="labels=__meta_type=infrastructure&labels=__meta_vendor=cisco&labels=__meta_region=AMER"

TARGET_GROUP5=network_switches_emea # SNMP via snmp_exporter
TARGET_GROUP5_HOSTS="10.20.1.101:60 10.20.1.102:60 10.20.1.103:60 10.20.1.104:60 10.20.1.105:60 10.20.1.106:60"
TARGET_GROUP5_LABELS="labels=__meta_type=infrastructure&labels=__meta_vendor=cisco&labels=__meta_region=EMEA"

TARGET_GROUP6=windows_server_metrics
TARGET_GROUP6_HOSTS="10.20.64.1:9199 10.20.64.2:9199 10.20.64.3:9199 10.20.64.4:9199 10.20.64.5:9199 10.20.64.6:9199 10.20.64.7:9199"
TARGET_GROUP6_LABELS="labels=__meta_type=infrastructure&labels=__meta_vendor=cisco&labels=__meta_region=EMEA"

# Delete all relevant Consul KV keys


# ----------------- TARGET GROUP 1 --------------------------

echo "Adding hosts to $TARGET_GROUP1 target group"
for H in ${TARGET_GROUP1_HOSTS}; do
  echo "localhost:80/api/target/${TARGET_GROUP1}/${H}"
  curl -XPOST "localhost:80/api/target/${TARGET_GROUP1}/${H}"
done
# Add labels to each target group
curl -XPOST "localhost:80/api/labels/update/${TARGET_GROUP1}?${TARGET_GROUP1_LABELS}"


# ----------------- TARGET GROUP 2 --------------------------
echo "Adding hosts to $TARGET_GROUP2 target group"
for H in ${TARGET_GROUP2_HOSTS}; do
  curl -XPOST "localhost:80/api/target/${TARGET_GROUP2}/${H}"
done
# Add labels to each target group
curl -XPOST "localhost:80/api/labels/update/${TARGET_GROUP2}?${TARGET_GROUP2_LABELS}"


# ----------------- TARGET GROUP 3 --------------------------

echo "Adding hosts to $TARGET_GROUP3 target group"
for H in ${TARGET_GROUP3_HOSTS}; do
  curl -XPOST "localhost:80/api/target/${TARGET_GROUP3}/${H}"
done
# Add labels to each target group
curl -XPOST "localhost:80/api/labels/update/${TARGET_GROUP3}?${TARGET_GROUP3_LABELS}"


# ----------------- TARGET GROUP 4 --------------------------

echo "Adding hosts to $TARGET_GROUP4 target group"
for H in ${TARGET_GROUP4_HOSTS}; do
  echo "localhost:80/api/target/${TARGET_GROUP4}/${H}"
  curl -XPOST "localhost:80/api/target/${TARGET_GROUP4}/${H}"
done
# Add labels to each target group
curl -XPOST "localhost:80/api/labels/update/${TARGET_GROUP4}?${TARGET_GROUP4_LABELS}"

# ----------------- TARGET GROUP 5 --------------------------

echo "Adding hosts to $TARGET_GROUP5 target group"
for H in ${TARGET_GROUP5_HOSTS}; do
  echo "localhost:80/api/target/${TARGET_GROUP5}/${H}"
  curl -XPOST "localhost:80/api/target/${TARGET_GROUP5}/${H}"
done
# Add labels to each target group
curl -XPOST "localhost:80/api/labels/update/${TARGET_GROUP5}?${TARGET_GROUP5_LABELS}"


# ----------------- TARGET GROUP 6 --------------------------

echo "Adding hosts to $TARGET_GROUP6 target group"
for H in ${TARGET_GROUP6_HOSTS}; do
  echo "localhost:80/api/target/${TARGET_GROUP6}/${H}"
  curl -XPOST "localhost:80/api/target/${TARGET_GROUP6}/${H}"
done
# Add labels to each target group
curl -XPOST "localhost:80/api/labels/update/${TARGET_GROUP6}?${TARGET_GROUP6_LABELS}"


# Delete labels from target group
# curl -XDELETE localhost:80/api/labels/update/nyz_webserver/__meta_host_type