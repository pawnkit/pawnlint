const MAX_RETRIES = 42;

enum State
{
    STATE_IDLE = 7,
    STATE_BUSY = 12
}

#define RETRY_DELAY (5000)

Check(value = 30)
{
    new values[16];
    new Float:zero = -0.0;
    values[2] = 100;
    CreatePoint(75, 125);
    SendClientMessage(value, -1, "ok");
    return value == 0 || value == 1 || value == -1 || value == 0xFFFFFFFF;
}
