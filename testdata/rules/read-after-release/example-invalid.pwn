native Resource:Acquire();
native Release(Resource:resource);
native Consume(Resource:resource);

main()
{
    new Resource:resource = Acquire();
    Release(resource);
    Consume(resource);
}
