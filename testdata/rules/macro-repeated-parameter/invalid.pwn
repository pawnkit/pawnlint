#define DOUBLE(x) ((x) + (x))
#define MAXIMUM(a, b) ((a) > (b) ? (a) : (b))
#define APPLY(%0) (Consume(%0) + Consume(%0))

main()
{
    new value = DOUBLE(NextValue());
}
