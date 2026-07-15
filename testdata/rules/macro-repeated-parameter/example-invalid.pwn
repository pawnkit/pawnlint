#define DOUBLE(%0) ((%0) + (%0))

main()
{
    new value = DOUBLE(random(10));
    return value;
}
