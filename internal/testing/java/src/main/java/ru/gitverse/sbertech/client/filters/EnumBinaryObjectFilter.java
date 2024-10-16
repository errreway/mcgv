package ru.gitverse.sbertech.client.filters;

import org.apache.ignite.binary.BinaryObject;
import org.apache.ignite.lang.IgniteBiPredicate;

public class EnumBinaryObjectFilter implements IgniteBiPredicate<Long, BinaryObject>  {
    private TestEnum.Enum val;

    public  EnumBinaryObjectFilter() {

    }

    public  EnumBinaryObjectFilter(TestEnum.Enum val) {
        this.val = val;
    }

    @Override
    public boolean apply(Long aLong, BinaryObject binaryObject) {
        return binaryObject != null && binaryObject.enumOrdinal() == val.ordinal();
    }
}
