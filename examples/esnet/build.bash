# set default configs
bin/disctl -update-config minicccfs /root/minicccfs
bin/disctl -update-config minirouterfs /root/minirouterfs

# load the latest ESNET topology
bin/ldesnet

# assign random UUIDs
bin/annotate uuid

# emit a minimega script
bin/minemiter
