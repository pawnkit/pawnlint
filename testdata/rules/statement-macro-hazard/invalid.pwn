#define TWO(x) First(x); Second(x)
#define TERMINATED(x) Work(x);
#define CONDITIONAL(x) if (x) Work()

main()
{
    TWO(1);
}
