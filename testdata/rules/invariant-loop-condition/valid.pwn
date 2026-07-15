new global_running;

Check(parameter)
{
    new remaining = 10;
    while (remaining > 0)
        remaining--;

    for (new index = 0; index < 10; index++)
        print("working");

    for (new item = 0, length = 10; item != length; ++item)
        print("multiple");

    for (new reverse = 10; --reverse != -1;)
        print("condition update");

    while (IsReady())
        print("waiting");

    while (global_running)
        print("global");

    while (parameter)
        print("parameter");

    new changed = 1;
    while (changed)
        Mutate(changed);

    new values[4];
    while (values[0])
        values[0]--;

    while (true)
        break;
}

#define UPDATE_VALUE(%0) ((%0) = 0)

CheckMacro()
{
    new value = 1;
    while (value)
        UPDATE_VALUE(value);
}
