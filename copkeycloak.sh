docker exec -it sihkaromicro-keycloak-1 \
  /opt/keycloak/bin/kc.sh export \
  --dir /tmp/export \
  --realm Clients \
  --users realm_file

docker cp sihkaromicro-keycloak-1:/tmp/export/Clients-realm.json ./keycloak/Clients-realm.json