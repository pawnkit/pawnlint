forward CountMismatch(value);

CountMismatch(value, extra)
{
}

forward Float:ReturnTag();

ReturnTag()
{
}

forward ParamTag(Float:value);

ParamTag(value)
{
}

forward ArrayRank(values[]);

ArrayRank(values)
{
}

forward Varargs(format[], ...);

Varargs(format[], value)
{
}

forward NameMismatch(value);

NameMismatch(other)
{
}

forward ConstMismatch(const value);

ConstMismatch(value)
{
}

forward ReferenceMismatch(&value);

ReferenceMismatch(value)
{
}

forward DimensionMismatch(values[2]);

DimensionMismatch(values[3])
{
}

forward DefaultMismatch(value = 1);

DefaultMismatch(value = 2)
{
}
