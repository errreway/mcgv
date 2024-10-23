package ru.gitverse.sbertech.client.filters;

import org.apache.ignite.binary.BinaryObject;
import org.apache.ignite.lang.IgniteBiPredicate;

public class TestEnumBinaryObjectFilter implements IgniteBiPredicate<Long, BinaryObject> {
    private TestEnum.Enum val;

    public TestEnumBinaryObjectFilter() {

    }

    public TestEnumBinaryObjectFilter(TestEnum.Enum val) {
        this.val = val;
    }

    @Override
    public boolean apply(Long aLong, BinaryObject binaryObject) {
        BinaryObject enumObj = binaryObject.<BinaryObject>field("enumField");
        BinaryObject[] enumArray = binaryObject.<BinaryObject[]>field("enumArrayField");
        if (enumObj != null && this.val != null && enumArray != null && enumArray.length > 0) {
            return enumObj.enumOrdinal() == this.val.ordinal() && enumArray[0].enumOrdinal() == this.val.ordinal();
        }
        return false;
    }
}
