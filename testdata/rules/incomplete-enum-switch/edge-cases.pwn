#include "shared-enum.inc"

enum Alias
{
    ALIAS_A = 1,
    ALIAS_B = 1,
    ALIAS_C = 2
}

AliasSwitch(Alias:value)
{
    switch (value)
    {
        case ALIAS_A:
            return 1;
        case ALIAS_C:
            return 2;
    }
    return 0;
}

UnknownCase(Alias:value)
{
    switch (value)
    {
        case UNKNOWN_VALUE:
            return 0;
    }
    return 1;
}

SharedSwitch(SharedState:value)
{
    switch (value)
    {
        case SHARED_IDLE:
            return 0;
    }
    return 1;
}
