DeepLoop(value)
{
    while (value)
    {
        if (value > 1)
        {
            for (new index = 0; index < value; index++)
            {
                value -= index;
            }
        }
    }
    return value;
}

DeepSwitch(value)
{
    switch (value)
    {
        case 1:
        {
            if (value)
            {
                do
                {
                    value--;
                }
                while (value);
            }
        }
    }
    return value;
}
