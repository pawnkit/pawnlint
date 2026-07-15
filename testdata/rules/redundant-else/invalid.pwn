ReturnValue(value)
{
    if (value < 0)
    {
        return 0;
    }
    else
    {
        return value;
    }
}

Nested(value)
{
    if (value == 0)
    {
        if (value < 0)
        {
            return -1;
        }
        else
        {
            goto done;
        }
    }
    else if (value > 0)
    {
        return 1;
    }

done:
    return 0;
}

Loop(values[], count)
{
    for (new index = 0; index < count; index++)
    {
        if (values[index] < 0)
            continue;
        else
            values[index]++;

        if (values[index] == 0)
            break;
        else
            values[index]--;
    }
}
