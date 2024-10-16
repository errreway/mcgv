package ru.gitverse.sbertech.client.filters;

public class TestEnum {
    private Long id;
    private Enum enumField;
    private Enum[] enumArrayField;

    public Long getId() {
        return id;
    }

    public void setId(Long id) {
        this.id = id;
    }

    public Enum getEnumField() {
        return enumField;
    }

    public void setEnumField(Enum enumField) {
        this.enumField = enumField;
    }

    public Enum[] getEnumArrayField() {
        return enumArrayField;
    }

    public void setEnumArrayField(Enum[] enumArrayField) {
        this.enumArrayField = enumArrayField;
    }

    public enum Enum {
        VAL1,
        VAL2,
        VAL3
    }
}
