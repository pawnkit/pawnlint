const MAX_LOGIN_ATTEMPTS = 3;

CanRetry(attempts)
{
    return attempts < MAX_LOGIN_ATTEMPTS;
}
