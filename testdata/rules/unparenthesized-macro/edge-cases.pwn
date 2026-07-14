#define NOARGS() 1 + 2
#define CALLSITE(x) foo(x)
#define SUBSCRIPT(x) arr[x]
#define WRAPPED_BODY_UNSAFE_PARAM(x) (x + x)
#define OBJECTLIKE_SAFE (1 + 2)
#define RETURN_STMT(x) return x

#if defined SOME_UNDEFINED_FEATURE
#define INSIDE_INACTIVE(x) x * x
#endif

main()
{
	new value = NOARGS();
}
