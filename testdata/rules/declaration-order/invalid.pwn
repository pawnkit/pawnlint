#include <core>

enum Values
{
    VALUE_FIRST
}

new globalValue;
const LATE_CONSTANT = 1;

main()
{
    print("start");
    new lateLocal;
    if (lateLocal)
    {
        print("nested");
        new nestedLate;
    }
    return lateLocal;
}
