# Nosee Sonde sites web

Sonde ayant vocation à surveiller des métrics sur un site web.
Temps de réponse, contenu du site, si le site est indexé, code HTTP.

### valeurs par défaut config :
NbRetentionsWarning = 2 
NbRetentionsCritical = 1

### Options
``` text
  -c string Duplicate old nosee sondes - abs path
  -w int Number of warning before alert
  -d string Directory with sondes
  -o string Destination directory for new toml files - abs path
  -t	Test mode - execute test part only
  -v	Print version
```
### Signaux écoutés
```text
USR1 : Va lire le dossier des sondes pour mettre à jour la liste des sondes.
USR2 : débug des sondes en cours avec des informations sur leurs satus
QUIT : renvoie la liste des go routines en cours
```
- kill -USR1 $(pidof nosee-web)
- kill -USR2 $(pidof nosee-web)

### Envs Requis
- SONDE_SLACK_WEBHOOK_URL
- SONDE_NOSEE_URL
- SONDE_NOSEE_INFLUXDB_URL
