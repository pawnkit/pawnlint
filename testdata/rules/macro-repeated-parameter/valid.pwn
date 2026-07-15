#define ADD_ONE(x) ((x) + 1)
#define ADD(a, b) ((a) + (b))
#define ARRAY_INFO(array) (sizeof(array) + (array)[0])
#define ARRAY_SHAPE(array) (sizeof(array) + tagof(array))
#define BUFFER(%0) format(%0, sizeof %0, "")
#define CONSTANT 5
#define NO_ARGS() (1 + 2)

main()
{
    new value = ADD_ONE(4);
}
