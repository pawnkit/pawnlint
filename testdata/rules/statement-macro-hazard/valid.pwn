#define WRAPPED(x) do { First(x); Second(x); } while (0)
#define BLOCK(x) { First(x); Second(x); }
#define COMPLETE(x) if (x) First(); else Second()
#define CALL(x) Work(x)
#define VALUE(x) ((x) + 1)
#define EMPTY()
#define DECLARE(name) forward name(); public name()

main()
{
    WRAPPED(1);
}
