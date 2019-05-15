delays="delay_SRC_A
delay_A_SRC
delay_A_B
delay_A_C
delay_B_A
delay_B_C
delay_B_D
delay_C_A
delay_C_B
delay_C_D
delay_D_B
delay_D_C
delay_D_DST
delay_DST_D"

mm() {
    echo "$@"
    /root/minimega -e $@ || exit $?
}

# set delays to random value from exponential distribution
for d in $delays; do
    mm .env $d $(python -c "import numpy; print(numpy.random.exponential())")ms
done

# launch model
mm read ospf-simple.mm
