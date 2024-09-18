package ru.gitverse.sbertech.client.filters;

import org.apache.ignite.binary.BinaryObject;
import org.apache.ignite.lang.IgniteBiPredicate;

public class PersonByNameBinaryObjectFilter implements IgniteBiPredicate<Long, BinaryObject> {
    private String name;

    public  PersonByNameBinaryObjectFilter() {

    }

    public PersonByNameBinaryObjectFilter(String name) {
        this.name = name;
    }

    @Override
    public boolean apply(Long aLong, BinaryObject binaryObject) {
        if (binaryObject != null) {
            var fName = binaryObject.field("name");
            return fName != null && fName.equals(name);
        }
        return false;
    }
}