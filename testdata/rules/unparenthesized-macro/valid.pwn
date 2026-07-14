#define SQUARE(x) ((x) * (x))
#define DOUBLE(x) ((x) + (x))
#define MAX(a, b) ((a) > (b) ? (a) : (b))
#define CALL(x) foo(x)
#define INDEX(x) arr[x]
#define ARRAY_LEN(arr) (sizeof(arr))
#define NEGATE(x) (-(x))
#define MAX_HP (100 + 50)
#define VERSION 5
#define GREETING "hello"

main()
{
	new value = SQUARE(1 + 2);
	new total = DOUBLE(value);
	new highest = MAX(value, total);
}
