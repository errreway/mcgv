package ru.gitverse.sbertech.client.filters;

import org.apache.ignite.lang.IgniteBiPredicate;

public class EnumArrayFilter implements IgniteBiPredicate<Long, Object[]> {
    private TestEnum.Enum val;

    public EnumArrayFilter() {

    }

    public EnumArrayFilter(TestEnum.Enum val) {
        this.val = val;
    }

    @Override
    public boolean apply(Long aLong, Object[] enums) {
        if (enums != null && enums.length > 0) {
            return enums[0] == val;
        }
        return false;
    }
}
