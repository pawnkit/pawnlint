enum Values
{
    VALUE_FIRST,
    VALUE_SECOND,
    VALUE_THIRD,
    VALUE_FOURTH
}

#if 0
new inactiveFirst, inactiveSecond, inactiveThird, inactiveFourth;
#endif

#if defined UNKNOWN_FEATURE
new uncertainFirst, uncertainSecond, uncertainThird, uncertainFourth;
#endif

new activeFirst;

UseValue(parameter)
{
    new local;
    return parameter + local + activeFirst + VALUE_FIRST;
}
