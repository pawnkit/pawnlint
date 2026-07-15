forward PublicRequired();

public PublicRequired()
{
    return 1;
}

#define INVOKE_MACRO_BOUNDARY() MacroBoundary()

forward MacroBoundary();

main()
{
    INVOKE_MACRO_BOUNDARY();
}

MacroBoundary()
{
    return 1;
}

#if UNKNOWN_FEATURE
forward UncertainForward();
UncertainForward()
{
    return 1;
}
#endif
