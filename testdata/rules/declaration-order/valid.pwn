#include <core>
#define VALUE (1)

enum Values
{
    VALUE_FIRST
}

const LIMIT = 10;
new globalValue;
native ExternalCall(value);
forward OnDeferred(value);

main()
{
    new localValue;
    localValue = ExternalCall(globalValue);
    return localValue;
}
