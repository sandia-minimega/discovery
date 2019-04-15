# wrapper that supresses output
disctl() {
    bin/disctl $@ > /dev/null || exit $?
}

# create 6 endpoints
for i in $(seq 1 6); do
    disctl -ne
    disctl -u nid=$i type qemu
done

# create 7 networks
for i in $(seq 1 7); do
    disctl -nn
done

# set names and router flags
disctl -u nid=1 name SRC
disctl -u nid=2 name A
disctl -u nid=2 router true
disctl -u nid=3 name B
disctl -u nid=3 router true
disctl -u nid=4 name C
disctl -u nid=4 router true
disctl -u nid=5 name D
disctl -u nid=5 router true
disctl -u nid=6 name DST

# connect endpoints and networks
disctl -c nid=7 nid=1
disctl -c nid=7 nid=2
disctl -c nid=8 nid=2
disctl -c nid=8 nid=3
disctl -c nid=9 nid=2
disctl -c nid=9 nid=4
disctl -c nid=10 nid=3
disctl -c nid=10 nid=4
disctl -c nid=11 nid=3
disctl -c nid=11 nid=5
disctl -c nid=12 nid=4
disctl -c nid=12 nid=5
disctl -c nid=13 nid=5
disctl -c nid=13 nid=6

# set IPs on all edges
disctl -u nid=1 n=7 ip 10.0.0.1/24
disctl -u nid=2 n=7 ip 10.0.0.2/24
disctl -u nid=2 n=8 ip 10.0.1.1/24
disctl -u nid=2 n=9 ip 10.0.2.1/24
disctl -u nid=3 n=8 ip 10.0.1.2/24
disctl -u nid=3 n=10 ip 10.0.3.1/24
disctl -u nid=3 n=11 ip 10.0.5.1/24
disctl -u nid=4 n=9 ip 10.0.2.2/24
disctl -u nid=4 n=10 ip 10.0.3.2/24
disctl -u nid=4 n=12 ip 10.0.4.1/24
disctl -u nid=5 n=11 ip 10.0.5.2/24
disctl -u nid=5 n=12 ip 10.0.4.2/24
disctl -u nid=5 n=13 ip 10.0.6.1/24
disctl -u nid=6 n=13 ip 10.0.6.2/24

# set default routes on SRC/DST
disctl -u nid=1 default_route 10.0.0.2
disctl -u nid=6 default_route 10.0.6.1

# enable OSPF on all router edges
disctl -u nid=2 n=7 OSPF true
disctl -u nid=2 n=8 OSPF true
disctl -u nid=2 n=9 OSPF true
disctl -u nid=3 n=8 OSPF true
disctl -u nid=3 n=10 OSPF true
disctl -u nid=3 n=11 OSPF true
disctl -u nid=4 n=9 OSPF true
disctl -u nid=4 n=10 OSPF true
disctl -u nid=4 n=12 OSPF true
disctl -u nid=5 n=11 OSPF true
disctl -u nid=5 n=12 OSPF true
disctl -u nid=5 n=13 OSPF true

# set parameterized delays on all edges
disctl -u nid=1 n=7 delay \$delay_SRC_A
disctl -u nid=2 n=7 delay \$delay_A_SRC
disctl -u nid=2 n=8 delay \$delay_A_B
disctl -u nid=2 n=9 delay \$delay_A_C
disctl -u nid=3 n=8 delay \$delay_B_A
disctl -u nid=3 n=10 delay \$delay_B_C
disctl -u nid=3 n=11 delay \$delay_B_D
disctl -u nid=4 n=9 delay \$delay_C_A
disctl -u nid=4 n=10 delay \$delay_C_B
disctl -u nid=4 n=12 delay \$delay_C_D
disctl -u nid=5 n=11 delay \$delay_D_B
disctl -u nid=5 n=12 delay \$delay_D_C
disctl -u nid=5 n=13 delay \$delay_D_DST
disctl -u nid=6 n=13 delay \$delay_DST_D

# set default kernels/initrds
disctl -update-config namespace rip-simple
disctl -update-config queueing true
disctl -update-config default_kernel file:miniccc.kernel
disctl -update-config default_initrd file:miniccc.initrd
disctl -update-config default_minirouter_kernel file:minirouter.kernel
disctl -update-config default_minirouter_initrd file:minirouter.initrd

# annotate nodes with uuids
bin/annotate uuid

# model ready!
bin/minemiter -w ospf-simple.mm
