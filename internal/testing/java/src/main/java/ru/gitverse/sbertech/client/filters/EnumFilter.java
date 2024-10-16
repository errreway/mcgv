package ru.gitverse.sbertech.client.filters;

import org.apache.ignite.lang.IgniteBiPredicate;

public class EnumFilter implements IgniteBiPredicate<Long, TestEnum.Enum> {
    private TestEnum.Enum val;

    public EnumFilter() {

    }

    public  EnumFilter(TestEnum.Enum val) {
        this.val = val;
    }

    @Override
    public boolean apply(Long aLong, TestEnum.Enum anEnum) {
        return anEnum == val;
    }
}
