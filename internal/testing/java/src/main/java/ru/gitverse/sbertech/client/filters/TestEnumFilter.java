package ru.gitverse.sbertech.client.filters;

import org.apache.ignite.lang.IgniteBiPredicate;

public class TestEnumFilter implements IgniteBiPredicate<Long, TestEnum> {
    private TestEnum.Enum val;

    public TestEnumFilter() {

    }

    public TestEnumFilter(TestEnum.Enum  val) {
        this.val = val;
    }

    @Override
    public boolean apply(Long aLong, TestEnum testEnum) {
        if (testEnum != null && testEnum.getEnumField() == val) {
            TestEnum.Enum[] arr = testEnum.getEnumArrayField();
            if (arr != null && arr.length > 0) {
                return arr[0] == val;
            }
            return true;
        }
        return false;
    }
}
