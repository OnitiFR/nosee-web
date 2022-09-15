# Nosee Sonde sites web

Sonde ayant a vocation a surveiller des métrics sur un site web.
Temps de réponse, contenu du site, si le site est indexé, code HTTP.

### Options
``` text
  -c string Duplicate old nosee sondes - abs path
  -d string Directory with sondes
  -o string Destination directory for new toml files - abs path
  -t	Test mode - execute test part only
  -v	Print version
```
### Signaux écoutés
```text
USR1 : débug des sondes en cours avec des informations sur leurs satus
USR2 : renvoie la liste des go routines en cours
```
kill -USR1 $(pidof go-sonde-wp)
kill -USR2 $(pidof go-sonde-wp)


### Envs Requis
SONDE_SLACK_WEBHOOK_URL
SONDE_NOSEE_URL
