##### DEFAULTS #####

# Set some global default configurations. These just tell discovery what
# container filesystems to use for the endpoint nodes and the router.
bin/disctl -update-config minicccfs /opt/minimega/images/miniccc
bin/disctl -update-config minirouterfs /opt/minimega/images/minirouter

##### BUILD GRAPH #####

# Create first network (we'll add two endpoints to it later). The value
# returned by the bin/disctl command is the ID of the network, which we'll
# need later to add endpoints to it.
net0=$(bin/disctl -nn)

# Create the two endpoints that will be connected to the first network.
# The value returned by the bin/disctl command is the ID of the endpoint,
# which we'll need later to add config options and add it to the
# network.

for i in $(seq 2); do
  # Create the endpoint.
  endpoint=$(bin/disctl -ne)

  # Add the endpoint to the first network.
  bin/disctl -c $net0 $endpoint
done

# Create second network (we'll add two endpoints to it later). The value
# returned by the bin/disctl command is the ID of the network, which we'll
# need later to add endpoints to it.
net1=$(bin/disctl -nn)

# Create the two endpoints that will be connected to the second network.
# The value returned by the bin/disctl command is the ID of the endpoint,
# which we'll need later to add config options and add it to the
# network.

for i in $(seq 2); do
  # Create the endpoint.
  endpoint=$(bin/disctl -ne)

  # Add the endpoint to the second network.
  bin/disctl -c $net1 $endpoint
done

# Create a router between the two networks that will serve up DHCP
# addresses to each network.

# Create the endpoint that will be the router.
router=$(bin/disctl -ne)

# Mark the endpoint as being a router.
bin/disctl -u $router router yes

# Set the name to use for the router so router configuration commands
# below know what name to use.
bin/disctl -u $router name rtr

# Connect the router to the first network.
bin/disctl -c $net0 $router

# Set the IP address to use for the edge connected to the first network.
bin/disctl -u $router N=$net0 ip 192.168.10.1/24

# Annotate the router to provide DHCP on the first edge we added above
# (ie. for the first network).
bin/disctl -u $router N=$net0 DHCP 192.168.10.0
bin/disctl -u $router N=$net0 DHCPLow 192.168.10.10
bin/disctl -u $router N=$net0 DHCPHigh 192.168.10.20
bin/disctl -u $router N=$net0 DHCPRouter 192.168.10.1

# Connect the router to the second network.
bin/disctl -c $net1 $router

# Set the IP address to use for the edge connected to the second network.
bin/disctl -u $router N=$net1 ip 192.168.20.1/24

# Annotate the router to provide DHCP on the second edge we added above
# (ie. for the second network).
bin/disctl -u $router N=$net1 DHCP 192.168.20.0
bin/disctl -u $router N=$net1 DHCPLow 192.168.20.10
bin/disctl -u $router N=$net1 DHCPHigh 192.168.20.20
bin/disctl -u $router N=$net1 DHCPRouter 192.168.20.1

##### ANNOTATE GRAPH #####

# Annotate graph, assigning random UUIDs.
bin/annotate uuid

##### EMIT GRAPH #####

# Emit minimega script.
bin/minemiter
