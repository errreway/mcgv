package ru.gitverse.sbertech.client.filters;

import org.apache.ignite.binary.BinaryObject;
import org.apache.ignite.lang.IgniteBiPredicate;

public class EnumArrayBinaryObjectFilter implements IgniteBiPredicate<Long, Object[]> {
    private TestEnum.Enum val;

    public EnumArrayBinaryObjectFilter() {

    }

    public  EnumArrayBinaryObjectFilter(TestEnum.Enum val) {
        this.val = val;
    }

    @Override
    public boolean apply(Long aLong, Object[] arr) {
        if (arr != null && arr.length > 0) {
            return arr[0] != null && ((BinaryObject)arr[0]).enumOrdinal() == val.ordinal();
        }
        return false;
    }
}
