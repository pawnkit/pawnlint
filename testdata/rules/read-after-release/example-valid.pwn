native Resource:Acquire();
native Release(Resource:resource);
native Consume(Resource:resource);

main()
{
    new Resource:resource = Acquire();
    Consume(resource);
    Release(resource);
}
