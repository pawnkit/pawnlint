forward RequiredForward(value);

main()
{
    return RequiredForward(1);
}

RequiredForward(value)
{
    return value;
}

forward ExternalCallback();

forward public ExportedByForward();

ExportedByForward()
{
    return 1;
}

forward AcrossInclude();
#include <other>
AcrossInclude()
{
    return 1;
}

forward Duplicate();
forward Duplicate();
Duplicate()
{
    return 1;
}
