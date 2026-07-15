CheckValue(value)
{
    if (value < 0)
    {
        value = 0;
    }
    else
    {
        value++;
    }

    if (value == 1)
    {
        if (value < 0)
            return 0;
    }
    else
    {
        value--;
    }
    return value;
}

#define RETURN_VALUE(%0) if (%0) return 1; else return 0
